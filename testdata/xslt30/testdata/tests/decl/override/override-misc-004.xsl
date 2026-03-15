<xsl:package 
  name="http://www.w3.org/xslt30tests/override-001"
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
    
  <!-- Keys in different packages can share the same name -->
  
  <xsl:key name="k" match="*" use="name()"/>
   
    <xsl:template name="main" visibility="public">
      <out>
        <a><xsl:copy-of select="$data/key('k', 'x')"/></a>
        <xsl:for-each select="$data">
          <xsl:call-template name="k"/>
        </xsl:for-each>
      </out>
    </xsl:template>
    
 
  
</xsl:package>   