<?xml version="1.0"?>

<!--
	XML Remote Processor Security Check Helper Stylesheet  
  Author:  Dimitre Novatchev (dnovatchev@yahoo.com)
-->

<xsl:stylesheet  version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xpath#axes?>
<!-- BUG: -->
<!-- DESCRIPTION -->

<xsl:output omit-xml-declaration="yes"/>

<xsl:variable name="root" select="/"/>

<xsl:template match="/" >

	<xsl:variable name="riskClasses">
		<language name="java">
			<class>java.io.File</class>
			<class>java.io.FileInputStream</class>
			<class>java.io.FileOutputStream</class>
			<class>java.io.RandomAccessFile</class>
			<class>java.io.FileReader</class>
		</language>
	</xsl:variable>
	
<xsl:variable name="Results"> 
	<xsl:for-each select="$riskClasses/language[@name='java']/class">
       
		<xsl:variable name="currentClass" select="."/>

		<xsl:for-each select="$root//*/namespace::*[contains(., $currentClass)]">
			<xsl:element name="Risk">
				<xsl:attribute name="keyWord"><xsl:value-of select="$currentClass"/></xsl:attribute>
			</xsl:element>
		</xsl:for-each>
	</xsl:for-each>
</xsl:variable>

<!-- This is the only output produced -->
<out><xsl:value-of select="count($Results/Risk)"/></out>

</xsl:template>

</xsl:stylesheet>

