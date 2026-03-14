<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering26 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Numbering comes from sorted order if value attribute used. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[1]/p[1]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/p[1]/text()[3]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[1]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(for-each)" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(sorting)" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:for-each select="chapter/note">
      <xsl:sort data-type="text" order="descending"/>
      <xsl:number format="-1-" value="position()"/>
      <xsl:value-of select="."/>
      <xsl:text>&#10;</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
