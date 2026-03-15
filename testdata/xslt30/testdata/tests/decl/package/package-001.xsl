<!-- the simplest possible package -->
<xsl:package
  name="http://www.w3.org/xslt30tests/package-001"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-001"
  exclude-result-prefixes="xs p">
  

    <xsl:template name="main" visibility="public">
      <ok/>
    </xsl:template>

</xsl:package>   