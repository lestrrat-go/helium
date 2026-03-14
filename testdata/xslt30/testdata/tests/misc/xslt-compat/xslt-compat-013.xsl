<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
  version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:ext="http://example.com/extensions"
  extension-element-prefixes="ext"
  exclude-result-prefixes="xs">
  
  <xsl:template match="/">
    <xsl:variable name="test" as="document-node(element(foo))?">
      <ext:instruction>
        <!-- Remove this and the error clears -->
        <xsl:fallback/>
      </ext:instruction>
    </xsl:variable>
    <xsl:if test="$test/foo[@bar]">
      <xsl:message>This is works fine</xsl:message>
    </xsl:if>
    <xsl:if test="$test/foo[@bar ne 'baz']">
      <xsl:message>This is works fine</xsl:message>
    </xsl:if>
    <xsl:if test="$test/foo[@bar eq 'baz']">
      <xsl:message>This throws "XPTY0020: Leading '/' cannot select the root node of the tree containing the context item: the context item is an atomic value"</xsl:message>
    </xsl:if>
    <xsl:if test="$test/foo[@bar = 'baz']">
      <xsl:message>This also throws "XPTY0020: Leading '/' cannot select the root node of the tree containing the context item: the context item is an atomic value"</xsl:message>
    </xsl:if>
    <!-- This also works fine -->
    <xsl:variable name="works" as="element()?">
      <ext:instruction>
        <xsl:fallback/>
      </ext:instruction>
    </xsl:variable>
    <xsl:if test="$works[@bar eq 'baz']">
      <xsl:message>This is works fine</xsl:message>
    </xsl:if>
    <a/>
  </xsl:template>
  
</xsl:stylesheet>