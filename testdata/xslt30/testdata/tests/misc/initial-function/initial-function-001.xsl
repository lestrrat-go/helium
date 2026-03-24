<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:saxon="http://saxon.sf.net/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:my="http://my.net/" exclude-result-prefixes="my saxon xs">

  <!-- PURPOSE test initial-function invocation -->

  <xsl:function name="my:foo" visibility="public">
    <xsl:param name="a"/>
    <xsl:param name="b"/>
    <simple-data><xsl:value-of select="$b"/>Simple Data<xsl:value-of select="$a"/></simple-data>
  </xsl:function>
  
  <xsl:function name="my:bar" as="item()*" visibility="public">
    <xsl:param name="a"/>
    <xsl:param name="b"/>
    <xsl:sequence select="$a,$b"/>
  </xsl:function>
  
  <xsl:function name="my:bad" as="item()*" visibility="public">
    <xsl:param name="a"/>
    <xsl:param name="b"/>
    <xsl:variable name="v">
      <x><xsl:copy-of select="$a"/></x>
    </xsl:variable>
    <xsl:apply-templates select="$v"/>
  </xsl:function>
  
  <xsl:template match="*">
    <xsl:result-document href="output.xml">
      <secondary-result/>
    </xsl:result-document>
  </xsl:template>

  <xsl:function name="my:private">
    <xsl:param name="a"/>
    <xsl:param name="b"/>
    <simple-data><xsl:value-of select="$b"/>Simple Data<xsl:value-of select="$a"/></simple-data>
  </xsl:function>

</xsl:stylesheet>
