<?xml version="1.0"?> 

<!-- test case for bug
     https://sourceforge.net/tracker/index.php?func=detail&aid=553347&group_id=29872&atid=397617
     (Context not reset on return from a call to a stylesheet function) -->

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xmi="http://ccfit.nsu.ru/gusarov">

  <xsl:strip-space elements="*"/>
  <xsl:template match="text() | @*"/>

  <xsl:variable name="actors" select="//Actor"/>

  <xsl:function name="xmi:name">
    <xsl:param name="this"/>
    <xsl:message>xmi:name</xsl:message>
    <xsl:sequence select="$this/name"/>
  </xsl:function>

  <xsl:function name="xmi:get_links_to_parents">
    <xsl:param name="this"/>
    <xsl:sequence select="$this/generalization/Gen/@xmi.idref"/>
  </xsl:function>

  <xsl:function name="xmi:get_links_to_children">
    <xsl:param name="this"/>
    <xsl:sequence select="$this/specialization/Gen/@xmi.idref"/>
  </xsl:function>

  <xsl:template match="/">
    <out>
      <section id="When we use functions and variables">
        <xsl:for-each select="$actors">
          <actor name="{name}">
            <children>
              <xsl:for-each
                select="$actors[xmi:get_links_to_parents(.) = xmi:get_links_to_children(current())]">
                <child>
                  <xsl:value-of select="name"/>
                </child>
              </xsl:for-each>
            </children>
            <parents>
              <xsl:for-each
                select="$actors[xmi:get_links_to_children(.) = xmi:get_links_to_parents(current())]">
                <parent>
                  <xsl:value-of select="name"/>
                </parent>
              </xsl:for-each>
            </parents>
          </actor>
        </xsl:for-each>
      </section>
      <section id="When we do not use functions">
        <xsl:for-each select="$actors">
          <actor name="{name}">
            <children>
              <xsl:for-each
                select="$actors[./generalization/Gen/@xmi.idref = current()/specialization/Gen/@xmi.idref]">
                <child>
                  <xsl:value-of select="name"/>
                </child>
              </xsl:for-each>
            </children>
            <parents>
              <xsl:for-each
                select="$actors[current()/generalization/Gen/@xmi.idref = ./specialization/Gen/@xmi.idref]">
                <parent>
                  <xsl:value-of select="name"/>
                </parent>
              </xsl:for-each>
            </parents>
          </actor>
        </xsl:for-each>
      </section>
      <section id="When we do not use variables">
        <xsl:for-each select="//Actor">
          <actor name="{name}">
            <children>
              <xsl:for-each
                select="//Actor[xmi:get_links_to_parents(.) = xmi:get_links_to_children(current())]">
                <child>
                  <xsl:value-of select="name"/>
                </child>
              </xsl:for-each>
            </children>
            <parents>
              <xsl:for-each
                select="//Actor[xmi:get_links_to_children(.) = xmi:get_links_to_parents(current())]">
                <parent>
                  <xsl:value-of select="name"/>
                </parent>
              </xsl:for-each>
            </parents>
          </actor>
        </xsl:for-each>
      </section>
    </out>
  </xsl:template>


</xsl:stylesheet>
