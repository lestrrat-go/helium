<xsl:package 
  name="http://www.w3.org/xslt30tests/override-misc-006"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <xsl:variable name="data"><e>x</e><x>y</x><g>z</g></xsl:variable>
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-901"
    package-version="1.0.0"/>
    
  <!-- Decimal formats in different packages can share the same name -->
  
  <xsl:decimal-format name="d" grouping-separator="!" decimal-separator=":"/>
   
    <xsl:template name="main" visibility="public">
      <xsl:for-each select="$data">
        <out>
          <a><xsl:value-of select="format-number(123456.78, '0!000:00', 'd')"/></a>
          <xsl:call-template name="d"/>         
        </out>
      </xsl:for-each>
    </xsl:template>
    
 
  
</xsl:package>   