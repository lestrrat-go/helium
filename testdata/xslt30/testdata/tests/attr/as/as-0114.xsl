<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable where @select contains an xs:float, xs:decimal or xs:integer,  
  				@as="xs:double".(type promotion)  -->

   <xslt:param name="var1" select="xs:float(-1.75e-3)" as="xs:double"/>

   <xslt:param name="var2" select="xs:decimal(-3.420100)" as="xs:double"/>

   <xslt:param name="var3" select="xs:integer(-300)" as="xs:double"/>

   <xslt:template match="/doc">
      <out>
         
         <var1>
            <xslt:value-of select="$var1 instance of xs:float"/>
            <xslt:value-of select="$var1 instance of xs:double"/>
         </var1>
         
         <var2>
            <xslt:value-of select="$var2 instance of xs:decimal"/>
            <xslt:value-of select="$var2 instance of xs:double"/>
         </var2>
         
         <var3>
            <xslt:value-of select="$var3 instance of xs:integer"/>
            <xslt:value-of select="$var3 instance of xs:double"/>
         </var3>
      </out>
   </xslt:template>
</xslt:transform>
