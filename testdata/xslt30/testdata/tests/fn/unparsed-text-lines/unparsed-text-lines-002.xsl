<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
<!-- Test unparsed-text-lines() function (XSLT 2.1) -->
<!-- Input file contains a trailing newline -->
   <?spec xslt#unparsed-text-lines?>

<xsl:template match="/" name="main">
    <out>
    <xsl:for-each select="unparsed-text-lines('quadruped-trailing-NL.txt', 'iso-8859-1')">
      <line><xsl:value-of select="."/></line>
    </xsl:for-each>
    </out>
</xsl:template>
       
</xsl:stylesheet>