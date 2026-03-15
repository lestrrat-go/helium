<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
                
                <!-- the first non-alpha token is a separator tokens if it appears
                between two formatting tokens -->

<xsl:template match="/">
<out>
 <xsl:for-each select="//c">
   <xsl:number count="*" level="multiple" format="(1)"/>
 </xsl:for-each>  
</out>
</xsl:template>

</xsl:stylesheet>