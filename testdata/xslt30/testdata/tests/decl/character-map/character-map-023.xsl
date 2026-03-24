<xsl:stylesheet version="2.0"  
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
 
 <?spec xslt#character-maps?>
 
 <xsl:output use-character-maps="root"/>
 
 <xsl:character-map name="root">
   <xsl:output-character character=">" string="GT"/> 
 </xsl:character-map>
 
 <xsl:template match="/">
 <a><xsl:copy-of select="root"/>
   <b att="&lt;&gt;&lt;&gt;"/>
 </a>
 </xsl:template>
 
</xsl:stylesheet> 
