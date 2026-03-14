<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" default-validation="strip">

  <!-- PURPOSE: Test validation on temporary tree using xsl:default-validation="preserve" on an LRE. -->
  <?spec xslt#creating-result-trees?>
  <?spec xslt#validating-document-nodes?>

  <xsl:variable name="tree" as="element()">
    <e xsl:default-validation="preserve">
      <e1>
        <xsl:attribute name="id" type="xs:ID">A001</xsl:attribute>
      </e1>
      <e2>
        <xsl:attribute name="id" type="xs:string">A002</xsl:attribute>
      </e2>
      <e3>
        <xsl:attribute name="id" type="xs:integer">003</xsl:attribute>
      </e3>
      <e4>
        <xsl:attribute name="id" type="xs:date">2002-10-20</xsl:attribute>
      </e4>
    </e>
  </xsl:variable>

  <xsl:template match="/">
    <out>
      <string>
        <xsl:value-of select="$tree//*[@id instance of attribute(*, xs:string)]/@id" separator=" "/>
      </string>
      <integer>
        <xsl:value-of select="$tree//*[@id instance of attribute(*, xs:integer)]/@id" separator=" "/>
      </integer>
      <date>
        <xsl:value-of select="$tree//*[@id instance of attribute(*, xs:date)]/@id" separator=" "/>
      </date>
    </out>
  </xsl:template>


</xsl:stylesheet>
