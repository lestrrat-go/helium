<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for resetting of an unspecified default namespace by a LRE. -->

<xsl:template match = "/">
  <out>
    <xsl:element namespace="http://www.bug.com/" name="foo">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


