<?xml version="1.0" encoding="UTF-8"?>

<!-- recursive stylesheet function test -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
	xmlns:DXFN="http://www.DXFN.com/" xmlns:UML="href://org.omg/UML/1.3"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:MS="http://www.ms.com">

	<xsl:function name="DXFN:node">
		<xsl:param name="id-ref"/>
		<xsl:sequence select="$root//*[@xmi.id = $id-ref]"/>
	</xsl:function>

	<xsl:function name="DXFN:logical-package-supplier-dependencies">
		<xsl:param name="node"/>
		<xsl:param name="offset"/>
		<xsl:variable name="v1" select=" $node[$offset]"/>
		<xsl:variable name="v2" select=" tokenize($v1/@clientDependency, ' ')"/>
		<xsl:variable name="this-node" select="DXFN:node( $v2)"/>
		<xsl:sequence
			select="if($offset &lt; count($node)) 
		            then $this-node | DXFN:logical-package-supplier-dependencies($node, $offset + 1 ) 
		            else $this-node"
		/>
	</xsl:function>

	<xsl:function name="DXFN:logical-package-hierarchy">
		<xsl:param name="package-node"/>
		<xsl:variable name="packages" select="$package-node/ancestor-or-self::UML:Package"/>
		<xsl:sequence select="$packages"/>
	</xsl:function>


	<xsl:variable name="root" select="/"/>

	<xsl:function name="MS:logical-package-dependency-suppliers-in-hierarchy">
		<xsl:param name="package-node-id"/>
		<xsl:sequence
			select="DXFN:logical-package-supplier-dependencies(
		                DXFN:logical-package-hierarchy(DXFN:node($package-node-id)), 1)"/>
	</xsl:function>

	<xsl:template match="/">
		<out>
			<xsl:for-each select="$root//UML:Class">
				<xsl:variable name="current-node" select="."/>


				<xsl:value-of
					select="count(MS:logical-package-dependency-suppliers-in-hierarchy(
				        $current-node/ancestor::UML:Package[1]/@xmi.id))"/>


			</xsl:for-each>
		</out>
	</xsl:template>

</xsl:stylesheet>
