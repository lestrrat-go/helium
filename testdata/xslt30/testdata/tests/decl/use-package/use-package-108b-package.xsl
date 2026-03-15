<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-008b"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:aliased="http://www.w3.org/xslt30tests/use-package-base-002"
  exclude-result-prefixes="xs p">
  
  <!-- tests that namespace aliases and xsl:output are local to a package -->
  
    <xsl:output use-character-maps="cm" name="test" omit-xml-declaration="yes"/>
  
    <xsl:character-map name="cm">
      <xsl:output-character character="z" string="ZZ"/>
    </xsl:character-map>
    
    <xsl:namespace-alias stylesheet-prefix="xs" result-prefix="aliased"/>
    
    <xsl:function name="p:alias" as="element()" visibility="public">
      <xs:test/>
    </xsl:function>
    
    <xsl:template name="go" visibility="public">
      <xsl:result-document format="test">
        <xs:out>zzz</xs:out>
      </xsl:result-document>
    </xsl:template>
    
  
  
</xsl:package>   