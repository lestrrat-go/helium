<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
   exclude-result-prefixes="xs f" version="3.0"  declared-modes="no" expand-text="yes">
   
   <xsl:variable name="module-uri" select="'http://www.w3.org/fots/fn/load-xquery-module/valid/module'"/>
   
   <xsl:template name="xsl:initial-template" visibility="public">
      <out>
         <xsl:variable name="module" select="load-xquery-module($module-uri)"/>
         <xsl:sequence select="$module?functions(QName($module-uri, 'func1'))?0()"/>
      </out>
   </xsl:template>
   
   
</xsl:package>