<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test of sequence-valued AVT in 2.0 mode. -->
  <xsl:template match="/">
    <out test="{1,2,3,4,5}" empty="{()}"/>
  </xsl:template>
 
</xsl:stylesheet>
