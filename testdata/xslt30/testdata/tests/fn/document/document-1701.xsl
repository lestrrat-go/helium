<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- Test unparsed-text() function (XSLT 2.0) -->
   <?spec xslt#unparsed-text?>

<xsl:template match="/">
    <out>
    <xsl:value-of select="replace(unparsed-text('xmdocs06.txt', 'iso-8859-1'), '\r\n', '&#xa;')"/>
    </out>
</xsl:template>
       
</xsl:stylesheet>