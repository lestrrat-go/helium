<xsl:package
  name="http://www.w3.org/xslt30-test/glob-cxt-item-010b"
  package-version="1.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">


    <xsl:global-context-item use="optional" as="xs:integer"/>
    
    <xsl:variable name="g" select="42"/>

          
  </xsl:package>