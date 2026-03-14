<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for resetting of the unspecified default namespace by a contained xsl:element. -->

<xsl:template match = "/">
  <out>
    <xsl:element namespace="http://abc.com/" name="foo">
      <xsl:element name="yyy"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


