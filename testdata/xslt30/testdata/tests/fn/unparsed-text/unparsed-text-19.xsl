<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- Test unparsed-text() function in a context where direct push to the serializer is possible -->
   <?spec xslt#unparsed-text?>

<xsl:template match="/">
    <out>
     <xsl:value-of select="unparsed-text('xmdocs06.txt', 'iso-8859-1')"/>
    </out>
</xsl:template>
       
</xsl:stylesheet>