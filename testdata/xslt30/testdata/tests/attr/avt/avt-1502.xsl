<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Check that attributes of top-level data elements are not treated as AVTs. -->
  <xsl:template match="doc"><x>OK</x></xsl:template>

  <my:data att="}" xmlns:my="http://my.uri/">
    <my:subdata att="}"/>
  </my:data>
 
</xsl:stylesheet>
