<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xxx="http://www.w3.org/1999/XSL/Transform"
  version="2.0">
  
  <!-- Prefixes in xsl:namespace-alias depend on the immediate namespace context -->
  
  <xsl:namespace-alias  stylesheet-prefix="foo" result-prefix="bar" xmlns:foo="http://foo1/" xmlns:bar="http://bar1"/>
  
  <xsl:namespace-alias  stylesheet-prefix="foo" result-prefix="bar" xmlns:foo="http://foo2/" xmlns:bar="http://bar2"/>
  
  <xsl:namespace-alias  stylesheet-prefix="foo" result-prefix="bar" xmlns:foo="http://foo3/" xmlns:bar="http://bar3"/>
  
  <xsl:template name="main">
    <out>
      <one xmlns="http://foo1/"/>
      <two xmlns="http://foo2/"/>
      <three xmlns="http://foo3/"/>
    </out>
  </xsl:template>
</xsl:stylesheet>