<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="http://www.w3.org/result-document-1101.ns" exclude-result-prefixes="f">
<!-- Purpose: Call to xsl:result-document is not allowed in temporary output state -->

   <xsl:template name="a">
	  <xsl:variable name="v">
	     <xsl:call-template name="rrr"/>
	  </xsl:variable>
	  <xsl:sequence select="$v"/>
   </xsl:template>
   
   <xsl:template name="b">
   <!-- Allowed in 3.0 -->
     <b>
	  <xsl:attribute name="v">
	     <xsl:call-template name="rrr"/>
	  </xsl:attribute>
	 </b>
   </xsl:template>
   
   <xsl:template name="c">
   <!-- Allowed in 3.0 -->
     <c>
	  <xsl:value-of>
	     <xsl:call-template name="rrr"/>
	  </xsl:value-of>
	 </c>
   </xsl:template>
   
   <xsl:template name="d">
   <!-- Allowed in 3.0 -->
     <d>
	  <xsl:comment>
	     <xsl:call-template name="rrr"/>
	  </xsl:comment>
	 </d>
   </xsl:template>
   
   <xsl:template name="e">
   <!-- Allowed in 3.0 -->
     <e>
	  <xsl:processing-instruction name="pi">
	     <xsl:call-template name="rrr"/>
	  </xsl:processing-instruction>
	 </e>
   </xsl:template>
   
   <xsl:template name="f">
   <!-- Allowed in 3.0 -->
     <f>
	  <xsl:namespace name="n">
	     <xsl:call-template name="rrr"/>
	  </xsl:namespace>
	 </f>
   </xsl:template>
   
   <xsl:template name="g">
     <g>
      <xsl:perform-sort select="1 to 100">      
	    <xsl:sort>
	     <xsl:call-template name="rrr"/>
	    </xsl:sort>
	  </xsl:perform-sort>  
	 </g>
   </xsl:template>
   
   <xsl:template name="h">
   <!-- Allowed in 3.0 -->
     <h>
	  <xsl:message>
	     <xsl:call-template name="rrr"/>
	  </xsl:message>
	 </h>
   </xsl:template>
   
   <xsl:template name="i">
     <i>
	  <xsl:call-template name="sss">
	     <xsl:with-param name="n">
	        <xsl:call-template name="rrr"/>
	     </xsl:with-param>   
	  </xsl:call-template>
	 </i>
   </xsl:template>
   
   <xsl:template name="j">
     <j>
	  <xsl:call-template name="ttt"/>
	 </j>
   </xsl:template>
   
   <xsl:key name="k" match="*">
     <xsl:call-template name="rrr"/>
   </xsl:key>  
   
   <xsl:template name="k">
     <xsl:variable name="v">
       <a>1</a>
       <b>2</b>
       <c>3</c>
     </xsl:variable>
     <k>
	  <xsl:sequence select="key('k', '2', $v)"/>
	 </k>
   </xsl:template>
   
   <xsl:template name="l">
     <xsl:sequence select="f:l()"/>
   </xsl:template>
   
   <xsl:function name="f:l">
     <xsl:call-template name="rrr"/>
   </xsl:function>
   
   <xsl:template name="m">
     <xsl:merge>
       <xsl:merge-source select="21 to 50">
         <xsl:merge-key select="string(.)"/>
       </xsl:merge-source>
       <xsl:merge-source select="10 to 40">
         <xsl:merge-key>
           <xsl:call-template name="rrr"/>
         </xsl:merge-key>
       </xsl:merge-source>
       <xsl:merge-action>3</xsl:merge-action>
     </xsl:merge>
   </xsl:template>
   
   <xsl:accumulator name="f:n" initial-value="0">
     <xsl:accumulator-rule match="*">
       <xsl:call-template name="rrr"/>
     </xsl:accumulator-rule>
   </xsl:accumulator>
   
   <xsl:template name="n">
     <xsl:variable name="doc"><a><b/></a></xsl:variable>
     <xsl:for-each select="$doc/a">
       <xsl:value-of select="accumulator-before('f:n')"/>
     </xsl:for-each>
   </xsl:template>      
   
   
   
   <xsl:template name="rrr">
     <xsl:result-document href="out.xml"><boo/></xsl:result-document>
     <xsl:text>rrr</xsl:text>
   </xsl:template>
   
   <xsl:template name="sss">
     <xsl:param name="n"/>
     <foo/>
   </xsl:template>
   
   <xsl:template name="ttt">
     <xsl:param name="n">
       <xsl:call-template name="rrr"/>
     </xsl:param>  
     <foo/>
   </xsl:template>
   
</xsl:transform>
