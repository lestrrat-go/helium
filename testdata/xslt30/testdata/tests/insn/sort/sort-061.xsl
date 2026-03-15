<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose:Sort a sequence of numeric values of various types that includes float NaNs and double NaNs. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="v1" select="xs:float(12.5)" as="xs:float"/>

   <xslt:variable name="v2" select="xs:integer(1)" as="xs:integer"/>

   <xslt:variable name="v3" select="xs:float('NaN')" as="xs:float"/>

   <xslt:variable name="v4" select=" xs:double('NaN')" as="xs:double"/>

   <xslt:variable name="v5" select="xs:float(0.009)" as="xs:float"/>

   <xslt:variable name="v6" select="xs:double(-0.05)" as="xs:double"/>

   <xslt:variable name="v7" select="xs:string(-0.00)" as="xs:string"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="($v1,$v2,$v3,$v4,$v5,$v6,$v7)">
            <xslt:sort select="." data-type="number" order="descending"/>
            <xslt:value-of select="."/>
            <xslt:value-of select="' | '"/>
         </xslt:for-each>
      </out>                                       
  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
