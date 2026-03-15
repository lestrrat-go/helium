<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for invalid namespace URI; spec says "not syntactically legal URI" is NOT error. -->

<xsl:template match = "doc">
  <out>
    <xsl:element name="ouch:foo" xmlns:ouch="&quot;">
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>