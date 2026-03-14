<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with complex pathExpr involving various node and name tests -->

   <t:variable name="var"> 
		    <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">6<num1>1<num2>2<num3>3</num3>
            </num2>
         </num1>
         <num4>4<num6>6</num6>
         </num4>
         <num5>5</num5>
      </doc>
	  </t:variable>

   <t:template match="doc">
		    <out>
         <t1>
            <t:choose>
	      			     <t:when test="name(../child::element()/attribute()[string(.) ne 'at2'][1]) = 'att1'">  
	        			      <t:for-each select="../child::element()/attribute()[string(.) ne 'at2']">
	          				      <t:value-of select="name(.)"/>
	          					     <t:text>|</t:text>
	        			      </t:for-each>
	     		 	     </t:when>
	      		 	    <t:otherwise>
	        			      <t:for-each select="reverse(../child::element()/attribute()[string(.) ne 'at2'])">
	          				      <t:value-of select="name(.)"/>
	          				      <t:text>|</t:text>
	        			      </t:for-each>
	      			     </t:otherwise>
	    		     </t:choose>
         </t1>
         <t2>
            <t:for-each select="($var/element()//child::*)[*]">
	    			       <t:value-of select="name(.)"/>
	    			       <t:text>|</t:text>
	   			     </t:for-each>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
