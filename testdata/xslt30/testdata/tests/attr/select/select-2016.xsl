<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:value-of with set operations: except, union and intersect -->

   <t:variable name="var"> 
		    <num1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">1<num2>2<num3>3</num3>
         </num2>
      </num1>
		    <num4 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">4<num6>6</num6>
      </num4>
		    <num5 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">5</num5>
	  </t:variable>

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="$var/* intersect $var//*" separator="|"/>
         </t1>
         <t2>
            <t:value-of select="$var//* except $var/*" separator="|"/>
         </t2>
         <t3>
            <t:value-of select="$var//* union $var/*" separator="|"/>
         </t3>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
