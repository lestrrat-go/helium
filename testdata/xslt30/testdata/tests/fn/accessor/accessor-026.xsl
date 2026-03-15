<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with copied nodes as argument -->

   <xslt:template match="/">
	
	     <xslt:variable name="var1">
	        <xslt:copy-of select="/"/>
      </xslt:variable>
  
	     <out>
	      <t1>
            <xslt:value-of select="if (data($var1/doc/processing-instruction()) instance of xs:string) then data($var1/doc/processing-instruction()) else 'WROONG TYPE returned'"/>
         </t1>
         <t2>
            <xslt:value-of select="if (data($var1/doc/comment()) instance of xs:string) then data($var1/doc/comment()) else 'WROONG TYPE returned'"/>
         </t2>
         <t3>
            <xslt:value-of select="if (data($var1/doc/text()[1]) instance of xs:untypedAtomic) then data($var1/doc/text()[1]) else 'WROONG TYPE returned'"/>
         </t3>
         <t4>
            <xslt:value-of select="if (data($var1/doc/element()) instance of xs:untypedAtomic) then data($var1/doc/element()) else 'WROONG TYPE returned'"/>
         </t4>
         <t5>
            <xslt:value-of select="if (data($var1/doc/attribute()) instance of xs:untypedAtomic) then data($var1/doc/attribute()) else 'WROONG TYPE returned'"/>
         </t5>
         <t6>
            <xslt:value-of select="if (data($var1) instance of xs:untypedAtomic) then data($var1) else 'WROONG TYPE returned'"/>
         </t6>
      </out>
   </xslt:template>
</xslt:transform>
