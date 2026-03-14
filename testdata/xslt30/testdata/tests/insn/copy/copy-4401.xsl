<xsl:transform version="2.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	
	<!-- Bug: Saxon-SA 8.7 gives different results from Saxon-B 8.7 -->

<xsl:variable name="link_fixup">
	<xsl:apply-templates/>
</xsl:variable>

<xsl:template match="*">
	<xsl:element name="{local-name()}" namespace="http://www.w3.org/1999/xhtml">
		<xsl:copy-of select="@*"/>
		<xsl:apply-templates/>
	</xsl:element>
</xsl:template>

<xsl:template match="a[@needsFixup]">
	<xsl:variable name="link">
		<a href="asdf" title="asdf" xmlns="http://www.w3.org/1999/xhtml">
			asdf
		</a>
	</xsl:variable>
	<xsl:copy-of select="$link"/>
</xsl:template>

<xsl:template match="/">
    <xsl:copy-of select="$link_fixup"/>
</xsl:template>

</xsl:transform>
