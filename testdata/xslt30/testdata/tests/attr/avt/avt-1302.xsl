<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Test of right curly brace escape. -->
  <xsl:template match="doc">
    <out test="}}"/>
  </xsl:template>
 
</xsl:stylesheet>
