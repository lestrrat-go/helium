<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Sort values that are cast to float in the select attribute and have some NaN values. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="doc/a">
            <xslt:sort select="xs:float(.)" data-type="number" order="descending"/>
            <xslt:value-of select="."/>
            <xslt:value-of select="' | '"/>
         </xslt:for-each>
      </out>                                       
  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
