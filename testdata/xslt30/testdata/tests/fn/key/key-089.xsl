<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="my"
  version="2.0" exclude-result-prefixes="xs">
  
  <xsl:function name="my:key" as="xs:string?">
    <xsl:param name="arg" as="xs:string?"/>
    <xsl:message select="'arg', $arg"/>
    <xsl:sequence select="$arg"/>
  </xsl:function>
  
  <xsl:key name="by-key" match="*[@key]" use="my:key(@key)"/>
  
  <xsl:template match="@* | *">
    <xsl:copy>
      <xsl:apply-templates select="@*, node()"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="key('by-key', 'c')"/>
</xsl:stylesheet>