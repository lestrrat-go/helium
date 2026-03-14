<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains built-in types derived by 
  				restriction from xs:decimal, @as="xs:float".(type promotion) Types tested:
				- xs:long, xs:nonNegativeInteger, xs:int, xs:byte-->

   <xslt:variable name="var1" select="/doc-schemaas/elem-int" as="xs:float"/>

   <xslt:variable name="var2" select="/doc-schemaas/elem-nonNegativeInteger" as="xs:float"/>

   <xslt:variable name="var3" select="/doc-schemaas/elem-long" as="xs:float"/>

   <xslt:variable name="var4" select="/doc-schemaas/elem-byte" as="xs:float"/>

   <xslt:template match="/doc-schemaas">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:int"/>
            <xslt:value-of select="$var1 instance of xs:float"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:nonNegativeInteger"/>
            <xslt:value-of select="$var2 instance of xs:float"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of xs:long"/>
            <xslt:value-of select="$var3 instance of xs:float"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of xs:byte"/>
            <xslt:value-of select="$var4 instance of xs:float"/>
         </var4>
      </out>
   </xslt:template>
</xslt:transform>
