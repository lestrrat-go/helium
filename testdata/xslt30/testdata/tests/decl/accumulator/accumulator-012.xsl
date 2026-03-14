<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum005/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Use accumulators to match text nodes -->
  
  <xsl:param name="streamable" static="yes" select="'no'"/>

  <xsl:accumulator name="f:last-p" 
  					as="xs:string" 
  					initial-value="''"
  					_streamable="{$streamable}">
    <xsl:accumulator-rule match="section/p/text()" select="string(.)"/>
  </xsl:accumulator>

  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-copy" use-accumulators="#all"/>

  <xsl:template match="section">
    <xsl:apply-templates/>
    <last><xsl:value-of select="accumulator-after('f:last-p')"/></last>
  </xsl:template>

  <xsl:strip-space elements="*"/>

</xsl:package>
