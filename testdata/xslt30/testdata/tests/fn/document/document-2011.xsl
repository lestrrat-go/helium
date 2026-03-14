<xsl:stylesheet exclude-result-prefixes="#all" version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   <xsl:output indent="yes" omit-xml-declaration="yes" />
   <xsl:template match="document-node()">
      <test prod="{system-property('xsl:product-version')}">
         <xsl:for-each select="., doc(document-uri(.))">
            <doc id="{generate-id()}" uri="{document-uri()}" />
         </xsl:for-each>
      </test>
   </xsl:template>
</xsl:stylesheet>