<?xml version="1.0" encoding="iso-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
                
<?spec xslt#conflict?>
<!--BUG 6.1/020: null pointer exception occurs when reporting
              ambiguous rule match -->

<xsl:template match="/">
 <doc>
   <xsl:apply-templates/>
 </doc>
</xsl:template>
                               
 <xsl:template match="*">
   <xsl:apply-templates/>
 </xsl:template>

<xsl:template match="@*|node()">
</xsl:template>

<xsl:template match="text()">
</xsl:template>

</xsl:stylesheet>
