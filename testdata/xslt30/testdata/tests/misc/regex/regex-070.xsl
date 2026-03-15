<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  
  <!-- PURPOSE: test replace() with special characters in replacement string;
      with and without use of q flag. -->
  
  <xsl:template match="/" mode="a">
    <xsl:param name="in" select="'Special characters?'"/>
    <xsl:param name="find" select="'?'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\?', '*$0*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\?', '\\$0'), '*$0*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '[\^\.\\\?\*\+\{\}\(\)\|\$\[\]]',
        '\\$0'), '*$0*')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="b">
    <xsl:param name="in" select="'Special characters?'"/>
    <xsl:param name="find" select="'?'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\?', '\$$0\$')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\?', '\\?'), '\$$0\$')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '?', '\?', 'q'), '\$$0\$')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="c">
    <xsl:param name="in" select="'Special characters\?'"/>
    <xsl:param name="find" select="'\?'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\\\?', '*$0')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\\\?', '\\\\\\?'), '*$0')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '\?', '\\\?', 'q'), '*$0')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="d">
    <xsl:param name="in" select="'Special characters$'"/>
    <xsl:param name="find" select="'$'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$', '\\$0\\')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$', '\\\$'), '\\$0\\')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '$', '\$', 'q'), '\\$0\\')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="e">
    <xsl:param name="in" select="'Special characters$`'"/>
    <xsl:param name="find" select="'$`'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$`', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$`', '\\\$`'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '$`', '\$`', 'q'), '*')"/></C>
      <D><xsl:value-of select="replace($in, replace($find, '$`', '\$`', 'q'), '\$''')"/></D>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="f">
    <xsl:param name="in" select="'Special characters$0'"/>
    <xsl:param name="find" select="'$0'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$0', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$0', '\\\$0'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '$0', '\$0', 'q'), '*')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="g">
    <xsl:param name="in" select="'Special characters$'"/>
    <xsl:param name="find" select="'$'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$', '\\$0'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '[\^\.\\\?\*\+\{\}\(\)\|\$\[\]]',
        '\\$0'), '*')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="h">
    <xsl:param name="in" select="'Special characters$0'"/>
    <xsl:param name="find" select="'$0'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$0', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$0', '\\$0'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '[\^\.\\\?\*\+\{\}\(\)\|\$\[\]]0',
        '\\$0'), '*')"/></C>
      <D><xsl:value-of select="replace($in, replace($find, '$0', '\$0', 'q'), '*')"/></D>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="i">
    <xsl:param name="in" select="'Special characters$0'"/>
    <xsl:param name="find" select="'$0'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$0', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '(\$0)', '\\$1'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '([\^\.\\\?\*\+\{\}\(\)\|\$\[\]]0)',
        '\\$1'), '*')"/></C>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="j">
    <xsl:param name="in" select="'Special characters$&amp;'"/>
    <xsl:param name="find" select="'$&amp;'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$&amp;', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$&amp;', '\\$0'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '[\^\.\\\?\*\+\{\}\(\)\|\$\[\]]&amp;',
        '\\$0'), '*')"/></C>
      <D><xsl:value-of select="replace($in, replace($find, '$&amp;', '\$&amp;', 'q'), '*')"/></D>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="k">
    <xsl:param name="in" select="'Special characters$$'"/>
    <xsl:param name="find" select="'$$'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\$\$', '*')"/></A>
      <B><xsl:value-of select="replace($in, replace($find, '\$', '\\$0'), '*')"/></B>
      <C><xsl:value-of select="replace($in, replace($find, '[\^\.\\\?\*\+\{\}\(\)\|\$\[\]]',
        '\\$0'), '*')"/></C>
      <D><xsl:value-of select="replace($in, replace($find, '$', '\$', 'q'), '*')"/></D>
    </out>
  </xsl:template>
  
  <xsl:template match="/" mode="l">
    <xsl:param name="in" select="'Special characters?'"/>
    <out>
      <A><xsl:value-of select="replace($in, '\?', '\$&amp;')"/></A>
      <B><xsl:value-of select="replace($in, '?', '$&amp;', 'q')"/></B>
      <C><xsl:value-of select="replace($in, '(\?)', '\\$1')"/></C>
      <D><xsl:value-of select="replace($in, '?', '$1', 'q')"/></D>
    </out>
  </xsl:template>
  
</xsl:stylesheet>