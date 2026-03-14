<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!-- Use xsl:copy-of within a function, which in Saxon will do a virtual copy -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:f="http://f/"
version="2.0">
    <xsl:template match="/">
      <xsl:variable name="in"><a/><b/><c/></xsl:variable>
      <a><xsl:value-of select="count($in/* | f:copy($in/*) | f:copy($in)/* )"/></a>
    </xsl:template>
    
    <xsl:function name="f:copy">
      <xsl:param name="node" as="node()*"/>
      <xsl:copy-of select="$node"/>
    </xsl:function>
</xsl:stylesheet>
