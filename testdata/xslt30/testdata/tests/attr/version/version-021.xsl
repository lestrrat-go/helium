<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Crashes with an NPE in Saxon 7.9.1 -->
<?spec xslt#backwards?>

<xsl:template match="/">
<a>
<xsl:value-of version="1.0">
   <xsl:value-of select="floor(nonexistent)"/>
</xsl:value-of>
</a>
</xsl:template>

</xsl:stylesheet>