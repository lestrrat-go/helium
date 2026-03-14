<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

<!-- Purpose: match zero-length substring. Allowed in 3.0 -->

   <xsl:variable name="v1" as="xs:string">Ten Thousand,10000,,"10,000","It's ""10 Grand"", mister",10K</xsl:variable>

   <xsl:template name="main" expand-text="yes">
      <out>        
        <xsl:analyze-string select="$v1" regex='(?:^|,)(?:"((?:[^"]|"")*)"|([^",]*))'>
          <xsl:matching-substring>
            <col>{replace(regex-group(1), '""', '"')||regex-group(2)}</col>
          </xsl:matching-substring>
        </xsl:analyze-string>
     </out>
   </xsl:template>
</xsl:transform>

<!-- '(?:^|,)(?:"((?:[^"]|"")*)"|([^",]*))"' -->