<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering91 -->
    <!-- Purpose: Test counting only some nodes, with id() in count pattern. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[1]/p[1]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[2]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[1]/p[1]/text()[1]" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="iddata">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="el">
  <xsl:choose>
    <xsl:when test="@id='b' or @id='d' or @id='g'">
      <xsl:number level="single" from="iddata" count="id('b d g')" format="(1) "/>
    </xsl:when>
    <xsl:otherwise>
      <xsl:text>(no) </xsl:text>
    </xsl:otherwise>
  </xsl:choose>
  <xsl:value-of select="@id"/>
</xsl:template>

</xsl:stylesheet>