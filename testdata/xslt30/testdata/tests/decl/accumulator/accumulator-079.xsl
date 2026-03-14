<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:mf="http://example.com/mf"
  xmlns:saxon="http://saxon.sf.net/"
  exclude-result-prefixes="#all" version="3.0">
  
  <xsl:param name="STREAMABLE" static="yes" select="false()"/>
  
  <xsl:mode on-no-match="shallow-skip" use-accumulators="#all" _streamable="{$STREAMABLE}"/>
  
  <xsl:accumulator name="A" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="accumulator-after('B') + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="B" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="accumulator-after('C') + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="C" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="$value + accumulator-after('D') + 1"/>
  </xsl:accumulator>
   
  <xsl:accumulator name="F" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="$value + accumulator-after('G') + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="G" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="$value + accumulator-after('values')"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="D" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="$value + accumulator-after('E') + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="E" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="fields/data/id1" phase="end"
      select="$value + accumulator-after('F') + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="values" as="xs:integer*" initial-value="0" _streamable="{$STREAMABLE}" saxon:trace="yes">
    <xsl:accumulator-rule match="data" select="0"/>
    <xsl:accumulator-rule match="data/id1/text() | data/id2/text()"
      select="xs:integer(.)"/>
  </xsl:accumulator>
  
  
  <xsl:template match="fields">
    <xsl:apply-templates/>
    <z><xsl:sequence select="accumulator-after('A')"/></z>
  </xsl:template>
  
  <xsl:template match="root">
    <out><xsl:apply-templates/></out>
  </xsl:template>
  
</xsl:stylesheet>