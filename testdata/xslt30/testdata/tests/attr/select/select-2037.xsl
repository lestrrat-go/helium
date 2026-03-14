<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with set operations -->

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
            <t:for-each select="$var/* intersect $var//*">
					          <t:value-of select="."/>
					          <t:text>|</t:text>
				        </t:for-each>
         </t1>
         <t2>
            <t:for-each select="$var//* except $var/*">
					          <t:value-of select="."/>
					          <t:text>|</t:text>
				        </t:for-each>
         </t2>
         <t3>
            <t:for-each select="$var//* union $var/*">
					          <t:value-of select="."/>
					          <t:text>|</t:text>
				        </t:for-each>
         </t3>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
