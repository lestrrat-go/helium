<?xml version="1.0"?> 

<!-- test use of id() in pattern with a variable (XSLT 2.0 feature) -->
<?spec xslt#patterns?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <xsl:template match="/">
        <d><xsl:apply-templates/></d>
    </xsl:template>
  
    <xsl:param name="x" select="'c'"/>
  
    <xsl:template match="a">[<xsl:value-of select="@id"/>]</xsl:template>
    <xsl:template match="id($x)">[*<xsl:value-of select="@id"/>*]</xsl:template>

    <xsl:template match="node()" ><xsl:apply-templates /></xsl:template>
</xsl:stylesheet>
