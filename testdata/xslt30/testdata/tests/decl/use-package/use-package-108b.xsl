<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-108"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:q="http://www.w3.org/xslt30tests/use-package-108"
  xmlns:principal="urn:wrong"
  exclude-result-prefixes="xs p q principal">

    <!-- tests that the output declaration and namespace alias are local to a package -->

    <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-008b" package-version="1.0.0"/>
    
    <xsl:output use-character-maps="cm" name="test" omit-xml-declaration="no"/>
    
    <xsl:character-map name="cm">
      <xsl:output-character character="z" string="XX"/>
    </xsl:character-map>

    <xsl:namespace-alias stylesheet-prefix="xs" result-prefix="principal"/>

    <xsl:template name="xsl:initial-template" visibility="public">
      <xsl:call-template name="go"/>
    </xsl:template>

</xsl:package>   