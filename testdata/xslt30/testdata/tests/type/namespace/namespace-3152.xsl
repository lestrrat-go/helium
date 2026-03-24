<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test crossing prefix set locally with namespace from outer level (where it has other prefix). -->

<xsl:template match = "/">
  <out xmlns:p1="http://xyz.com/">
    <xsl:element namespace="http://xyz/" name="p2:foo" xmlns:p2="http://other.com/">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>