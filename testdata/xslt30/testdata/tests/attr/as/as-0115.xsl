<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable where @select contains an xs:decimal or xs:integer, 
  				@as="xs:float".(type promotion)  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var2" select="xs:decimal(-3.420100)" as="xs:float"/>

   <xslt:variable name="var3" select="xs:integer(-300)" as="xs:float"/>

   <xslt:template match="/doc">
      <out>         
         <var2>
            <xslt:value-of select="$var2 instance of xs:decimal"/>
            <xslt:value-of select="$var2 instance of xs:float"/>
         </var2>
         
         <var3>
            <xslt:value-of select="$var3 instance of xs:integer"/>
            <xslt:value-of select="$var3 instance of xs:float"/>
         </var3>
      </out>
   </xslt:template>
</xslt:transform>
