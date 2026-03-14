<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Test of sequence-valued AVT in backwards compatibility mode. -->
  <xsl:template match="/">
    <out test="{1,2,3,4,5}" empty="{()}"/>
  </xsl:template>
 
</xsl:stylesheet>
