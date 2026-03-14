<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:anamespace="http://foo.com/" exclude-result-prefixes="anamespace">

  <!-- Purpose: Test xsl:exclude-result-prefixes, stylesheet level -->

  <xsl:template match="/">
    <out>
      <p>
        <xsl:attribute name="test" namespace="http://foo.com/">true</xsl:attribute>
      </p>
    </out>
  </xsl:template>

</xsl:stylesheet>
