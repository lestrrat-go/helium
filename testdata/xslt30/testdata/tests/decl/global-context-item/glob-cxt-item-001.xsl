<xsl:package
  name="http://www.w3.org/xslt30-test/glob-cxt-item-001"
  package-version="1.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <xsl:mode/>

    <xsl:global-context-item use="required" as="document-node(element(doc))"/>
    
    <xsl:variable name="g" select="/doc"/>

    <xsl:template match="/">
      <out>
         <xsl:copy-of select="$g"/>
      </out>
    </xsl:template>  
          
  </xsl:package>