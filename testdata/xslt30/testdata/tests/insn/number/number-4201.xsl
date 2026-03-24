<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:z="http://z.test.com/">

  <!-- CaseName: numbering86 -->
    <!-- Purpose: Namespaced elements should work just like non-namespaced ones. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[1]/p[1]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[2]/p[1]/text()[3]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[3]" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="z:doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="z:note">
  <xsl:element name="znote">
    <xsl:attribute name="number">
      <xsl:number level="single" from="z:chapter"/>
    </xsl:attribute>
    <xsl:value-of select="."/>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
