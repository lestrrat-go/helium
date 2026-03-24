<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:sch002="http://ns.example.com/sch002" exclude-result-prefixes="sch002" version="2.0">

  <!-- Purpose: to be imported into other stylesheets -->


  <xsl:variable name="v1" as="sch002:testType*">
    <xsl:sequence
      select="(sch002:testType(2000),sch002:testType(2001),sch002:testType(2002),sch002:testType(2003))"
    />
  </xsl:variable>

  <xsl:variable name="v2" as="element()">
    <sch002:complexTest xsl:type="sch002:complexTestType" sch002:type="333">
      <subject>math</subject>
      <size>50</size>
    </sch002:complexTest>
  </xsl:variable>

  <xsl:variable name="v3" as="attribute()">
    <xsl:attribute name="sch002:type" type="sch002:testType">444</xsl:attribute>
  </xsl:variable>

  <xsl:template match="/doc">
    <out>
      <type>
        <xsl:value-of select="sum($v1)"/>
      </type>
      <elem>
        <xsl:value-of select="$v2 instance of schema-element(sch002:complexTest)"/>
        <xsl:text> * </xsl:text>
        <xsl:value-of select="$v2/@sch002:type instance of schema-attribute(sch002:type)"/>
        <xsl:text> * </xsl:text>
        <xsl:copy-of select="$v2"/>
      </elem>
      <attrib>
        <xsl:value-of select="$v3 instance of schema-attribute(sch002:type)"/>
        <xsl:text> * </xsl:text>
        <xsl:value-of select="$v3"/>
      </attrib>
    </out>
  </xsl:template>

</xsl:stylesheet>
