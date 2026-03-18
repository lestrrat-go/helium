<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:transform version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    
    <!-- test unparsed-text-available() -->
   <?spec xslt#unparsed-text?>
<xsl:output name="test" encoding="iso-8859-1" indent="yes"/>
<xsl:template match="/">
  <xsl:result-document format="test">
  <out>
    <a><xsl:value-of select="unparsed-text-available('http://www.w3.org/Consortium/mission.html')"/></a>
    <a><xsl:value-of select="unparsed-text-available('stdxmlfile.xml', 'iso-8859-1')"/></a>   
    <a><xsl:value-of select="unparsed-text-available('porridge.xml')"/></a>    
    <a><xsl:value-of select="unparsed-text-available('custard.xml', 'iso-646')"/></a>                                    
  </out> 
  </xsl:result-document> 
</xsl:template>


</xsl:transform>