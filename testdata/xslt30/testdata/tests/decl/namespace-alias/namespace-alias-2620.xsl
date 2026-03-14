<xsl:stylesheet
  version="2.0" 
  xmlns="http://www.w3.org/1999/XSL/Transform"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:x="http://www.example.org/alias"
  xmlns:y="http://www.example.org/My/Transform/Language">

  <xsl:import href="namespace-alias-2620a.xsl" />

  <xsl:namespace-alias stylesheet-prefix="x" result-prefix="y"/>

  <xsl:template match="/">
    <x:transform />
  </xsl:template>

</xsl:stylesheet>