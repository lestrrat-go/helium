<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with node stored in variable -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:variable name="var1" select="processing-instruction()"/>
            <xslt:value-of select="if (data($var1) instance of xs:string) then data($var1) else 'WROONG TYPE returned'"/>
         </t1>
         <t2>
            <xslt:variable name="var2" select="comment()"/>
            <xslt:value-of select="if (data($var2) instance of xs:string) then data($var2) else 'WROONG TYPE returned'"/>
         </t2>
         <t3>
            <xslt:variable name="var3" select="text()[1]"/>
            <xslt:value-of select="if (data($var3) instance of xs:untypedAtomic) then data($var3) else 'WROONG TYPE returned'"/>
         </t3>
         <t4>
            <xslt:variable name="var4" select="element()"/>
            <xslt:value-of select="if (data($var4) instance of xs:untypedAtomic) then data($var4) else 'WROONG TYPE returned'"/>
         </t4>
         <t5>
            <xslt:variable name="var5" select="attribute()"/>
            <xslt:value-of select="if (data($var5) instance of xs:untypedAtomic) then data($var5) else 'WROONG TYPE returned'"/>
         </t5>
         <t6>
            <xslt:variable name="var6" select="/"/>
            <xslt:value-of select="if (data($var6) instance of xs:untypedAtomic) then data($var6) else 'WROONG TYPE returned'"/>
         </t6>
         <t7>
            <xslt:variable name="var7" select="namespace::*"/>
            <xslt:for-each select="$var7">
			            <xslt:value-of select="if (data($var7) instance of xs:string) then data($var7) else 'WROONG TYPE returned'"/>
		          </xslt:for-each>
         </t7>
      </out>
   </xslt:template>
</xslt:transform>
