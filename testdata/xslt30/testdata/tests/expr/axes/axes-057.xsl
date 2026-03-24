<?xml version="1.0"?> 

  <!-- PURPOSE: Test all axes with a "simple text" temporary tree. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:my="http://my.uri/" exclude-result-prefixes="my">

  <xsl:template match="/">

    <xsl:variable name="t">text node</xsl:variable>

    <xsl:variable name="n0" select="$t"/>
    <xsl:variable name="n1" select="$t/text()"/>

    <xsl:variable name="id0" select="generate-id($t)"/>
    <xsl:variable name="id1" select="generate-id($t/text())"/>
    <out>
      <parent>
        <xsl:copy-of select="my:compare($n0/self::node(), $id0)"/>
        <xsl:copy-of select="my:compare($n0/parent::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/child::node(), $id1)"/>
        <xsl:copy-of select="my:compare($n0/descendant::node(), $id1)"/>
        <xsl:copy-of select="my:compare($n0/descendant-or-self::node(), concat($id0,':',$id1))"/>
        <xsl:copy-of select="my:compare($n0/ancestor::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/ancestor-or-self::node(), $id0)"/>
        <xsl:copy-of select="my:compare($n0/preceding-sibling::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/following-sibling::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/preceding::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/following::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/attribute::node(), '')"/>
        <xsl:copy-of select="my:compare($n0/namespace::node(), '')"/>
      </parent>
      <child>
        <xsl:copy-of select="my:compare($n1/self::node(), $id1)"/>
        <xsl:copy-of select="my:compare($n1/parent::node(), $id0)"/>
        <xsl:copy-of select="my:compare($n1/child::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/descendant::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/descendant-or-self::node(), $id1)"/>
        <xsl:copy-of select="my:compare($n1/ancestor::node(), $id0)"/>
        <xsl:copy-of select="my:compare($n1/ancestor-or-self::node(), concat($id0,':',$id1))"/>
        <xsl:copy-of select="my:compare($n1/preceding-sibling::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/following-sibling::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/preceding::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/following::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/attribute::node(), '')"/>
        <xsl:copy-of select="my:compare($n1/namespace::node(), '')"/>
      </child>
    </out>
  </xsl:template>

  <xsl:function name="my:compare">
    <xsl:param name="nodes"/>
    <xsl:param name="required-id"/>
    <xsl:variable name="actual-id">
      <xsl:value-of select="for $n in $nodes return generate-id($n)" separator=":"/>
    </xsl:variable>
    <xsl:choose>
      <xsl:when test="$required-id = $actual-id">
        <ok/>
      </xsl:when>
      <xsl:otherwise>
        <error required="{$required-id}" actual="{$actual-id}"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:function>
</xsl:stylesheet>
