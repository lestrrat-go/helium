<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering24 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Number without value= inside sorted for-each. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[1]/p[1]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/p[2]/text()[3]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[1]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(for-each)" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(sorting)" -->
  <!-- Scenario: operation="standard-XML" -->
  <!-- Elaboration: Numbering directly inside sorted for-each is from document order,
     if using default number calculations. "If no value attribute is specified,
     then the xsl:number element inserts a number based on the position of the
     current node in the source tree." -->

<xsl:template match="doc">
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:for-each select="chapter/note">
      <xsl:sort data-type="text" order="descending"/>
      <xsl:number format="-1-" level="single" from="chapter" /><xsl:value-of select="."/>
      <xsl:text>&#10;</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
