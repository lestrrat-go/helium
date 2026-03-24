<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    
    <xsl:template match="/">
      <out>
        <xsl:call-template name="ws"/>
      </out>
    </xsl:template>

    <xsl:template name="ws" xml:space="preserve">  <xsl:context-item as="document-node()"/>  </xsl:template>
          
  </xsl:stylesheet>