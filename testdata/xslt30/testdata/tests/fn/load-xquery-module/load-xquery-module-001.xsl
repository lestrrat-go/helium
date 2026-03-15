<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no" expand-text="yes">
 
  <xsl:template name="xsl:initial-template" visibility="public">
    <xsl:variable name="module" select="load-xquery-module('unknown-query')"/>
    <xsl:sequence select="$module?variables(xs:QName('magic'))"/>
  </xsl:template>
  

</xsl:package>
