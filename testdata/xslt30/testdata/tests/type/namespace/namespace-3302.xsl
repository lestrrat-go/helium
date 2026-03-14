<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Basic test of applying namespaces to elements. -->

  <xsl:template match="/">
    <out xmlns:anamespace="http://foo.com/">
      <xsl:element name="test" namespace="http://foo.com/">
        <inner/>
      </xsl:element>
      <later/>
      <anamespace:anelement/>
    </out>
  </xsl:template>

</xsl:stylesheet>
