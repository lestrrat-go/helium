<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:param name="streamable" as="xs:boolean" static="true" select="false()"/>
  
  <xsl:mode _streamable="{$streamable}" use-accumulators="#all"/>
  
  <xsl:accumulator name="a1" as="xs:string" initial-value="'init'" streamable="yes">
    <xsl:accumulator-rule match="document-node()" select="$value || ', ' || 'matched root /'"/>
    <xsl:accumulator-rule match="*" select="$value || ', ' || 'matched element /' || ancestor-or-self::node()/node-name() => string-join('/')"/>
  </xsl:accumulator>
  
  <xsl:template match="/" priority="2">
    <out>
      <xsl:next-match/>
    </out>
  </xsl:template>
  
  <xsl:template match="/ | *">
    <processed node="/{ancestor-or-self::node()/node-name() => string-join('/')}" value="{accumulator-before('a1')}"/>
    <xsl:apply-templates mode="#current"/>
  </xsl:template>
  
</xsl:stylesheet>
