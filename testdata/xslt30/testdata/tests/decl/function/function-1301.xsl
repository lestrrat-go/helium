<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:f="http://fxsl.sf.net/"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:str-reverse-func="http://fxsl.sf.net/str-reverse-func"
 exclude-result-prefixes="f str-reverse-func xs"
>

<!-- Test of str:reverse() function from Dimitre Novatchev's FXSL library -->

  <xsl:output method="xml" encoding="iso-8859-1"/>
  <!-- To be applied on text.xml -->
  
  <xsl:template match="/">
    <out><xsl:value-of select="f:str-reverse(/*)"/></out>
  </xsl:template>
  
  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
    </xsl:apply-templates>
  </xsl:function>


  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   <xsl:param name="arg9"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
      <xsl:with-param name="arg9" select="$arg9"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   <xsl:param name="arg9"/>
   <xsl:param name="arg10"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
      <xsl:with-param name="arg9" select="$arg9"/>
      <xsl:with-param name="arg10" select="$arg10"/>
    </xsl:apply-templates>
  </xsl:function>
  
      <xsl:function name="f:str-foldl">
      <xsl:param name="pFunc" as="element()"/>
      <xsl:param name="pA0"/>
      <xsl:param name="pStr" as="xs:string"/>

    <xsl:sequence select=
      "if(not($pStr))
         then
            $pA0
         else
            for $vcntList in string-length($pStr) return
               if($vcntList = 1)
                 then
                    f:apply($pFunc, $pA0, substring($pStr,1,1))
                 else
                  (:   for $vHalfLen in floor($vcntList div 2),
                         $fold in f:str-foldl($pFunc, 
                                              $pA0, 
                                              substring($pStr, 1, $vHalfLen)
                                              ) return
                      f:str-foldl($pFunc, 
                                  $fold,
                                  substring($pStr,$vHalfLen+1)
                                  ) :)
                   for $vHalfLen in floor($vcntList div 2) return
                      f:str-foldl($pFunc, 
                                  f:str-foldl($pFunc, 
                                              $pA0, 
                                              substring($pStr, 1, $vHalfLen)
                                              ),
                                  substring($pStr,$vHalfLen+1)
                                  ) "
      />
    </xsl:function>
    
    <xsl:function name="f:str-reverse">
      <xsl:param name="pStr" as="xs:string"/>

      <xsl:variable name="vReverseFun" 
                    select="document('')/*/str-reverse-func:*[1]"/>

      <xsl:sequence select="f:str-foldl($vReverseFun, (), $pStr)"/>
    </xsl:function>

    <str-reverse-func:str-reverse-func/>
    <xsl:template match="str-reverse-func:*" mode="f:FXSL">
         <xsl:param name="arg1" select="0"/>
         <xsl:param name="arg2" select="0"/>

         <xsl:value-of select="concat($arg2, $arg1)"/>
    </xsl:template>    
    
</xsl:stylesheet>