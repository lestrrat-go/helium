<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: Test xsl:supports-higher-order-functions -->

   <xsl:template match="/">
      <out>
         <xsl:value-of select="(system-property('xsl:supports-higher-order-functions') = 'yes' and function-available('filter', 2)) 
            or (system-property('xsl:supports-higher-order-functions') = 'no' and not(function-available('filter', 2)))"/>
      </out>
   </xsl:template>
</xsl:transform>
