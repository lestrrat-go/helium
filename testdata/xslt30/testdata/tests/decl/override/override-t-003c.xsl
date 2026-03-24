<xsl:package 
  name="http://www.w3.org/xslt30tests/override-103"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-101"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-102"
    package-version="1.0.0">
    
    <xsl:accept component="template" names="t-abstract" visibility="hidden"/>
  
    <!-- empty override is allowed, but only if all abstract components are made hidden. A dynamic error is raised if nevertheless the abstract component is called -->
    <xsl:override>
    </xsl:override>
    
    
  </xsl:use-package>
  
  
  
    <!-- Demonstrate that the named template u in the base package is rebound to the declaration of t in this package -->
   
  <xsl:template name="main" visibility="public">
    <out>
      <xsl:call-template name="u">
            <xsl:with-param name="in" select="'x'"/>
            <xsl:with-param name="count" select="5"/>
      </xsl:call-template>
    </out>  
  </xsl:template>
    
  
  
</xsl:package>   