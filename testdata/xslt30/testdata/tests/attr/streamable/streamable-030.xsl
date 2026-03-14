<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">


  <!-- within a streaming template, use copy-of with multiple elements and attributes -->

  <xsl:mode streamable="yes"/>

  <xsl:output method="xml" indent="yes" encoding="UTF-8"/>

  <xsl:strip-space elements="*"/>

  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="mixed.xml">
        <xsl:apply-templates select="."/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <xsl:template match="coverpg|preface|titlepg"/>

  <xsl:template match="*">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>

  <xsl:template match="book">
    <book>
      <xsl:variable name="temp" as="node()*">
        <xsl:for-each select=".//*">
          <xsl:copy select="."/>
          <xsl:copy-of select="@*"/>
        </xsl:for-each>  
        <!--<xsl:sequence select=".//*/(copy-of(.), @*/copy-of(.))    (: was .//*|.//@* :)"/>-->
      </xsl:variable>
      <elements>
        <xsl:value-of select="count($temp[. instance of element()])"/>
      </elements>
      <attributes>
        <xsl:value-of select="count($temp[. instance of attribute()])"/>
      </attributes>
      <ok1 r="{$temp[1] instance of element(bktlong)}"/>
      <ok2 r="{$temp[2] instance of element(bktshort)}"/>
      <ok3 r="{$temp[3] instance of element(chapter)}"/>
      <ok4 r="{$temp[4] instance of attribute()}"/>
      <ok5 r="{$temp[5] instance of attribute()}"/>
      <ok6 r="{$temp[6] instance of element(chtitle)}"/>
      <ok7 r="{$temp[7] instance of element(v)}"/>
      <ok8 r="{$temp[8] instance of element(deity)}"/>
    </book>
  </xsl:template>

</xsl:transform>
