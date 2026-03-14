<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test for positional indexing in sequence  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="values" as="xs:integer*">
      <xslt:sequence select="(1,2,3,4)"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <in>
            <xslt:value-of select="$values[0]"/>
         </in>
         <in>
            <xslt:value-of select="$values[1]"/>
         </in>
         <in>
            <xslt:value-of select="$values[4]"/>
         </in>
         <in>
            <xslt:value-of select="$values[5]"/>
         </in>
      </out>
   </xslt:template>
</xslt:transform>
